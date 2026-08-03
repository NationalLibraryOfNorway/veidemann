import {Component, inject, provideZonelessChangeDetection} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {Ability, createMongoAbility, MongoAbility} from '@casl/ability';
import {AbilityServiceSignal} from '@casl/angular';

@Component({
  template: `
    @if (can('read', 'dashboard')) {
      <span data-testid="allowed">Allowed</span>
    }
  `,
  standalone: true,
})
class AbilitySignalHostComponent {
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  protected readonly can = this.abilityService.can;
}

describe('AbilityServiceSignal', () => {
  it('updates the template when ability rules change', async () => {
    const ability = createMongoAbility();

    await TestBed.configureTestingModule({
      imports: [AbilitySignalHostComponent],
      providers: [
        provideZonelessChangeDetection(),
        {provide: Ability, useValue: ability},
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(AbilitySignalHostComponent);
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[data-testid="allowed"]')).toBeNull();

    ability.update([{action: 'read', subject: 'dashboard'}]);
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[data-testid="allowed"]')).not.toBeNull();

    ability.update([]);
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[data-testid="allowed"]')).toBeNull();
  });
});
