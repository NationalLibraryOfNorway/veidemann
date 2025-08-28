import {ResourceComponent} from './resource.component';
import {CoreTestingModule} from '../../../core/core.testing.module';
import {Resource} from '../../../shared/models';
import {CommonsModule} from '../../../commons';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideZonelessChangeDetection } from '@angular/core';

describe('ResourceComponent', () => {
  let fixture: ComponentFixture<ResourceComponent>;
  let component: ResourceComponent;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CoreTestingModule.forRoot(),
        CommonsModule,
      ],
      declarations: [ResourceComponent],
      providers: [provideZonelessChangeDetection()]
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
