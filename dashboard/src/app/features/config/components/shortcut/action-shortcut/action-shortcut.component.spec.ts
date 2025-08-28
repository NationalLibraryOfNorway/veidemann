import {ActionShortcutComponent} from './action-shortcut.component';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {ConfigObject, Kind} from '../../../../shared/models';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatListModule} from '@angular/material/list';

describe('ActionShortcutComponent', () => {
  let component: ActionShortcutComponent;
  let fixture: ComponentFixture<ActionShortcutComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CoreTestingModule.forRoot(), MatListModule],
      declarations: [ActionShortcutComponent],
      providers: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ActionShortcutComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject();
    await fixture.whenStable();
  });

  it('should create', async () => {
    component.configObject = new ConfigObject();
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('should create with SEED', async () => {
    component.configObject = new ConfigObject({kind: Kind.SEED});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });


  it('should create with CRAWLJOB', async () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLJOB});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });


  it('should create with CRAWLENTITY', async () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLENTITY});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });


});
