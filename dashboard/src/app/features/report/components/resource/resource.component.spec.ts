import {ResourceComponent} from './resource.component';
import {Resource} from '../../../../shared/models';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {provideNoopAnimations} from '@angular/platform-browser/animations';

describe('ResourceComponent', () => {
  let fixture: ComponentFixture<ResourceComponent>;
  let component: ResourceComponent;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ResourceComponent],
      providers: [
        ...provideCoreTesting,
        provideNoopAnimations()
      ]
    });

    fixture = TestBed.createComponent(ResourceComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });


  it('should create with resource', async () => {
    const resource = new Resource(
      {
        uri: 'www.example.com',
        discoveryPath: 'dilldall'
      });

    component.resources = [resource];
    await fixture.whenStable();

    expect(component).toBeTruthy();
  });

});
